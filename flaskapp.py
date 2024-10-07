from flask import Flask, render_template, request
import yfinance as yf
import matplotlib.pyplot as plt
import mplfinance as mpf
from time import sleep
from pystyle import Colors 
import os
from vaderSentiment.vaderSentiment import SentimentIntensityAnalyzer
import pandas as pd
import datetime


app = Flask(__name__)

df = None
ticker = ""
@app.route('/')
def index():
    return render_template('index.html')

@app.route('/process', methods=['POST'])
def process():
    global ticker
    inputData = request.form['input_data']
    ticker = inputData
    try:
     calcstock(ticker)
     return render_template('stocks.html')
     
    except Exception as e:
        return f'Error: {e}'

def calculatePeRatio(price_per_share, earnings_per_share):
    try:
        peRatio = price_per_share / earnings_per_share
        return peRatio
    except ZeroDivisionError:
        return None

def determineValuation(peRatio, industryAverage):
    if peRatio is None:
        return "Cannot determine valuation due to zero earnings per share"
    if peRatio < industryAverage:
        return "Undervalued"
    elif peRatio > industryAverage:
        return "Overvalued"
    else:
        return "Fairly valued"

def doStockCalc(pricePerShare,earningsPerShare,industryAverage):
    try:
        peRatio = calculatePeRatio(pricePerShare, earningsPerShare)
        valuation = determineValuation(peRatio, industryAverage)
        if peRatio is not None:
            return valuation
        else:
            print("Cannot determine valuation due to zero earnings per share.")
    
    except ValueError:
        print("Invalid input. Please enter valid numeric values.")

def calcstock(ticker):
    global df
    stock = yf.Ticker(ticker)
    stockInf = stock.info
    print(stockInf)
    if 'longName' in stockInf:
        long_name = stockInf['longName']
        print(f"{Colors.cyan}Long Name: {long_name}")
    marketColorsDict = {
        'candle': {
            'up': 'g',
            'down': 'r',
        },
        'edge': {
            'up': 'k',
            'down': 'k',
        },
        'wick': {
            'up': 'k',
            'down': 'k',
        },
        'ohlc': {
            'up': 'k',
            'down': 'k',
        },
        'volume': {
            'up': '#1f77b4',
            'down': '#1f77b4',
        },
        'vcedge': {
            'up': '#1f77b4',
            'down': '#1f77b4',
        },
        'alpha': 1.0,  
    }
    
    
    def stockImageGen(endDate, startDate, daysbefore, period, long_name, marketColors):
        try:
            df = stock.history(period=period, start=startDate, end=endDate)
            df['PriceChange'] = df['Close'].diff()
            colors = ['g' if price_change >= 0 else 'r' for price_change in df['PriceChange']]
            plt.switch_backend('agg')  
            s = mpf.make_mpf_style(marketcolors=marketColors)
            mpf.plot(df, type='candle', style=s, title=f'{long_name} Stock Price', ylabel='Price', volume=False)
            plt.savefig(f'static/images/stockimage_{daysbefore}.png')  # Making the website actually look good ??????????? just saving the plots here
            plt.close()
            print("Candlestick chart saved successfully.")
        except Exception as e:
             print(f'Error {e}')
             return f'Error: {e}'
             
    currentDate = datetime.datetime.now()
    formattedDate = currentDate.strftime("%Y-%m-%d")
    endDate = formattedDate
    
    start_date = currentDate - datetime.timedelta(days=365)
    formatted_date = start_date.strftime("%Y-%m-%d")
    start_date = formatted_date
    df = stock.history(period='1d', start=start_date, end=endDate)

    stockImageGen(endDate, (currentDate-datetime.timedelta(days=365)).strftime("%Y-%m-%d"), '1y', '1d', long_name, marketColorsDict)
    stockImageGen(endDate, (currentDate-datetime.timedelta(days=7)).strftime("%Y-%m-%d"), '1w', '1h', long_name, marketColorsDict)
    stockImageGen(endDate, (currentDate-datetime.timedelta(days=30)).strftime("%Y-%m-%d"), '1m', '1h', long_name, marketColorsDict)
    stockImageGen(endDate, (currentDate-datetime.timedelta(days=1)).strftime("%Y-%m-%d"), '1d', '20m', long_name, marketColorsDict)
    
    shortMovingAvg = df['Close'].rolling(window=20).mean()
    longMovingAvg = df['Close'].rolling(window=50).mean()



    long_name = ''
    current_price = ''
    day_high = ''
    trailing_eps = ''
    fiftyTwoWeekHigh = ''
    fiftyTwoWeekLow = ''
    dayLow = ''
    previousClose = ''

    if 'currentPrice' in stockInf:  #appending it all to the files
        current_price = stockInf['currentPrice']
        print(f"Current Price: {current_price}")

    if 'dayHigh' in stockInf:
        day_high = stockInf['dayHigh']
        print(f"Day High: {day_high}")
    if 'trailingEps' in stockInf:
        trailing_eps = stockInf['trailingEps']
        print(f"Trailing Eps: {trailing_eps}")
    if 'fiftyTwoWeekHigh' in stockInf:
        fiftyTwoWeekHigh = stockInf['fiftyTwoWeekHigh']
        print(f"fiftyTwoWeekHigh: {fiftyTwoWeekHigh}")
    
    if 'fiftyTwoWeekLow' in stockInf:
        fiftyTwoWeekLow = stockInf['fiftyTwoWeekLow']
        print(f"fiftyTwoWeekLow: {fiftyTwoWeekLow}")

    if 'dayLow' in stockInf:
        dayLow = stockInf['dayLow']
        print(f"Day Low: {dayLow}")
    if 'previousClose' in stockInf:
        previousClose = stockInf['previousClose']
        print(f"previous Close: {previousClose}")
    if 'longName' in stockInf:
        long_name = stockInf['longName']
        print(f"{Colors.cyan}Long Name: {long_name}")
    valuation = doStockCalc(current_price, trailing_eps, calculatePeRatio(current_price, trailing_eps))
    print(f'The stock is {valuation}.')
    
    #Graph 2

    
    plt.clf()
    plt.plot(df.index, shortMovingAvg, label='20-Day MA', color='orange', alpha=0.7) #we got fried by the end of the hackathon
    plt.plot(df.index, longMovingAvg, label='50-Day MA', color='blue', alpha=0.7)
    plt.title(f'{long_name} Stock Price and Moving Averages')

    plt.savefig('static/images/movingaverage.png', dpi=100) 

    with open('values.txt', 'w') as file:
        info = f"Name: {long_name}\nCurrentPrice: {current_price}\nStockValuation: {valuation}\nDayHigh: {day_high}\nDayLow: {dayLow}\nPreviousClose: {previousClose}\nFiftyTwoWeekHigh: {fiftyTwoWeekHigh}\nFiftyTwoWeekLow: {fiftyTwoWeekLow}"
        file.write(info)


    analyzer = SentimentIntensityAnalyzer()
    news = stock.news 

    scores = []
    for item in news:
        text = item['title']
        sentiment = analyzer.polarity_scores(text)
        compound_sentiment = sentiment['compound']
        website_name = item['link']  
        scores.append((text, website_name, compound_sentiment))

    
    df = pd.DataFrame(scores, columns=['article_title', 'website_name', 'sentiment_rating'])
    file_path = 'sentiment_analysis.csv'
    if os.path.exists(file_path):
     os.remove(file_path)
    df.to_csv('sentiment_analysis.csv', index=False)

if __name__ == '__main__':
     app.run(debug=True)


